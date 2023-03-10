using System;
using System.Collections;
using System.Collections.Generic;
using System.Text;
using System.Threading.Tasks;
using Google.Protobuf;
using Google.Protobuf.Collections;
using Grpc.Core;
using protoBuff;
using UnityEngine;

//todo add try catches in places to get errors
// todo make sure that there is a connection / detect disconnect
namespace Online
{
    public class NetworkManager : MonoBehaviour
    {
        public GameObject[] spawnables;
        private Dictionary<string, GameObject> _spawnables;

        public int updateFps = 60; // update at 60 fps

        private readonly Dictionary<ByteString, NetworkedElement> _objects;
        private readonly Dictionary<ByteString, (Vector3, Quaternion)> _objectLastPos;

        private delegate void RunOnMainthread();

        private readonly Queue<RunOnMainthread> _mainthreadQueue;

        public NetworkManager()

        {
            _objects = new Dictionary<ByteString, NetworkedElement>();
            _objectLastPos = new Dictionary<ByteString, (Vector3, Quaternion)>();
            _mainthreadQueue = new Queue<RunOnMainthread>();
            GRPC.RegisterMessageCallback(onMessage);
        }

        // Start is called before the first frame update
        public void Start()
        {
            // kill self if other instances of object exist
            var others = FindObjectsOfType<NetworkManager>();
            foreach (var other in others)
            {
                if (other.gameObject == gameObject) continue;
                return;
            }

            _spawnables = new Dictionary<string, GameObject>();
            foreach (var spawnable in spawnables)
            {
                var networkedElement = spawnable.GetComponent<NetworkedElement>();
                if (networkedElement == null)
                    throw new Exception(spawnable.name + " is missing a script that implements NetworkedElement");

                if (_spawnables.ContainsKey(networkedElement.ID()))
                    throw new Exception("name collision with " + networkedElement.ID());
                _spawnables[networkedElement.ID()] = spawnable;
            }

            DontDestroyOnLoad(gameObject);

            Connect("The Only One");
        }

        public void Update()
        {
            while (_mainthreadQueue.Count > 0)
            {
                try
                {
                    _mainthreadQueue.Dequeue()();
                }
                catch (MissingReferenceException e)
                {
                }
            }
        }

        /// be careful with this and dont have scripts register on wake since it can lead to recursion 
        public void RegisterObject(NetworkedElement obj)
        {
            var id = Guid.NewGuid().ToByteArray();
            var uid = ByteString.CopyFrom(id);
            _objects.Add(uid, obj);
            PostRegistration(uid, obj);
        }

        public void UnregisterObject(NetworkedElement obj)
        {
            var id=ByteString.Empty;
            foreach (var (uid, element) in _objects)
            {
                if (!element.Equals(obj)) continue;
                id = uid;
                break;
            }

            UnregisterObject(id);
        }

        public void UnregisterObject(ByteString id)
        {
            if (_objects.ContainsKey(id)) return;
            _objects[id].Destroy();
            _objects.Remove(id);

            var req = new StreamAction
            {
                RemoveEntity = new RemoveEntity
                {
                    Id = id
                }
            };
            GRPC.SendRequest(req);
        }

        public async Task<bool> Connect(string sessionID)
        {
            RepeatedField<Entity> entities;
            try
            {
                entities = await GRPC.Connect(sessionID);
            }
            catch (RpcException e)
            {
                if (e.StatusCode == StatusCode.Unknown) Debug.LogWarning(e.Status.Detail);
                return false;
            }

            Debug.Log(entities);

            foreach (var entity in entities)
            {
                AddEntity(entity);
            }

            try
            {
                GRPC.StartStream();
            }
            catch (RpcException e)
            {
                if (e.StatusCode == StatusCode.Unknown) Debug.LogWarning(e.Status.Detail);
                return false;
            }

            PostRegistrers();

            StartCoroutine(UpdatePosition());
            return true;
        }

        private void onMessage(Response action)
        {
            foreach (var response in action.Responses) // maybe unwrap events and fire them one by one
            {
                // Debug.Log(action);
                RunOnMainthread function = null;
                switch (response.ActionCase)
                {
                    case StreamAction.ActionOneofCase.AddEntity:
                        function = () => { AddEntity(response.AddEntity.Entity); };
                        break;
                    case StreamAction.ActionOneofCase.RemoveEntity:
                        function = () => { RemoveEntity(response.RemoveEntity.Id); };
                        break;
                    case StreamAction.ActionOneofCase.UpdateEntity:
                        function = () => { UpdateEntity(response.UpdateEntity.Entity); };
                        break;
                    case StreamAction.ActionOneofCase.MoveEntity:
                        function = () => { MoveEntity(response.MoveEntity); };
                        break;
                    case StreamAction.ActionOneofCase.None:
                    default:
                        break;
                }

                if (function != null)
                    _mainthreadQueue.Enqueue(function);
            }
        }

        public void UpdateObject(NetworkedElement obj)
        {
            var objectID = ByteString.Empty;
            foreach (var (id, element) in _objects)
            {
                if (element != obj) continue;
                objectID = id;
                break;
            }

            if (objectID == ByteString.Empty) throw new Exception("Cant update, not registered");

            var pos = obj.GetPosition();
            GRPC.SendRequest(new StreamAction
            {
                UpdateEntity = new UpdateEntity
                {
                    Entity = new Entity
                    {
                        Data = obj.Data(),
                        Id = objectID,
                        Position = Helpers.ToPosition(pos.Item1),
                        Rotation = Helpers.ToRotation(pos.Item2),
                        Type = obj.ID()
                    }
                }
            });
        }

        private bool isControlled(ByteString id)
        {
            return _objects.ContainsKey(id) && _objects[id].GetControlType() == ElementType.Owner;
        }

        private void AddEntity(Entity entity)
        {
            if (_objects.ContainsKey(entity.Id)) return;
            var factory = new GameObject().AddComponent<Factory>();
            var script = factory.SpawnElement(entity, _spawnables[entity.Type]);
            _objects[entity.Id] = script;
        }

        private void RemoveEntity(ByteString id)
        {
            if (isControlled(id)) return;
            var obj = _objects[id];
            _objects.Remove(id);
            obj.Destroy();
        }

        private void UpdateEntity(Entity entity)
        {
            if (isControlled(entity.Id)) return;
            _objects[entity.Id]
                .HandleUpdate(Helpers.ToVector3(entity.Position), Helpers.ToQuaternion(entity.Rotation), entity.Data);
        }

        private void MoveEntity(MoveEntity moveAction)
        {
            if (isControlled(moveAction.Id)) return;
            _objects[moveAction.Id].HandleUpdate(Helpers.ToVector3(moveAction.Position),
                Helpers.ToQuaternion(moveAction.Rotation), "");
        }

        private void OnDestroy()
        {
            Disconnect();
        }

        private void OnApplicationQuit()
        {
            Disconnect();
        }

        [RuntimeInitializeOnLoadMethod]
        static void RunOnStart()
        {
            Application.quitting += GRPC.Disconnect;
            Application.wantsToQuit += () =>
            {
                GRPC.Disconnect();
                Connection.Dispose(); //todo make task that starts that will quit the app and a progress bar
                var state = Connection.GetChannelState();
                return state == ChannelState.Shutdown || state == ChannelState.TransientFailure;
            };
        }

        public async void Disconnect()
        {
            StopAllCoroutines();
            await Task.Delay((int)(1000f / updateFps) + 10);
            GRPC.Disconnect();
        }

        private void PostRegistrers()
        {
            foreach (var (id, obj) in _objects)
            {
                if (obj.GetControlType() == ElementType.Owner)
                    PostRegistration(id, obj);
            }
        }

        private void PostRegistration(ByteString id, NetworkedElement obj)
        {
            var pos = obj.GetPosition();
            var req = new StreamAction
            {
                AddEntity = new AddEntity
                {
                    KeepOnDisconnect = !obj.RemoveOnDisconnect(),
                    Entity = new Entity
                    {
                        Id = id,
                        Type = obj.ID(),
                        Data = obj.Data(),
                        Position = Helpers.ToPosition(pos.Item1),
                        Rotation = Helpers.ToRotation(pos.Item2)
                    }
                }
            };
            GRPC.SendRequest(req);
        }

        IEnumerator UpdatePosition()
        {
            while (true)
            {
                var requests = new RepeatedField<StreamAction>();
                foreach (var (id, element) in _objects)
                {
                    if (element.GetControlType() == ElementType.Listener) continue;
                    // ideally projectiles should be controlled by the server but i am making them be controlled by the sender for simplicities sake

                    (Vector3, Quaternion) pos;
                    try
                    {
                        pos = element.GetPosition();
                    }
                    catch (MissingReferenceException e)
                    {
                        continue; // object was destroyed
                    }

                    if (_objectLastPos.ContainsKey(id) &&
                        _objectLastPos[id] == pos) continue;
                    _objectLastPos[id] = pos;

                    requests.Add(new StreamAction
                    {
                        MoveEntity = new MoveEntity
                        {
                            Id = id,
                            Position = Helpers.ToPosition(pos.Item1),
                            Rotation = Helpers.ToRotation(pos.Item2)
                        }
                    });
                }

                if (requests.Count > 0)
                    GRPC.SendRequest(new Request { Requests = { requests } });

                yield return new WaitForSeconds(1f / updateFps);
            }
        }
    }
}