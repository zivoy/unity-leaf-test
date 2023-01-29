using System;
using System.Collections;
using System.Collections.Generic;
using System.Threading.Tasks;
using Google.Protobuf.Collections;
using Grpc.Core;
using UnityEngine;
using protoBuff;
using Request = protoBuff.Request;

//todo add try catches in places
// todo change to calls and responses layout
namespace Online
{
    public class NetworkManager : MonoBehaviour
    {
        public GameObject[] spawnables;
        private Dictionary<string, GameObject> _spawnables;

        public int updateFps = 60; // update at 60 fps

        private readonly Dictionary<string, NetworkedElement> _objects;
        private readonly Dictionary<string, Vector2> _objectLastPos;


        public NetworkManager()
        {
            _objects = new Dictionary<string, NetworkedElement>();
            _objectLastPos = new Dictionary<string, Vector2>();
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
                    throw new Exception(spawnable.name + " is missing an script that implements NetworkedElement");

                if (_spawnables.ContainsKey(networkedElement.ID()))
                    throw new Exception("name collision with " + networkedElement.ID());
                _spawnables[networkedElement.ID()] = spawnable;
            }

            DontDestroyOnLoad(gameObject);

            Connect();
        }

        /// be careful with this and dont have scripts register on wake since it can lead to recursion 
        public void RegisterObject(NetworkedElement obj)
        {
            var id = Guid.NewGuid().ToString();

            _objects.Add(id, obj);
        }

        public void UnregisterObject(NetworkedElement obj)
        {
            var id = "";
            foreach (var keyValuePair in _objects)
            {
                if (!keyValuePair.Value.Equals(obj)) continue;
                id = keyValuePair.Key;
                break;
            }
            UnregisterObject(id);
        }
        public void UnregisterObject(string id)
        {
            if (_objects.ContainsKey(id)) return;
            _objects[id].Destroy();
            _objects.Remove(id);

            var req = new Request
            {
                RemoveEntity = new RemoveEntity
                {
                    Id = id
                }
            };
            GRPC.SendRequest(req);
        }

        public void Connect()
        {
            RepeatedField<Entity> entities;
            try
            {
                entities = GRPC.Connect("The Only One");
            }
            catch (RpcException e)
            {
                if (e.StatusCode == StatusCode.Unknown) Debug.LogWarning(e.Status.Detail);
                return;
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
                return;
            }

            PostRegistrars();

            StartCoroutine(UpdatePosition());
        }

        //todo implement the rest of player connection, make sure that there is a connection / detext disconnect, work out the dispose as well, its not leaving session

        private void onMessage(Response action)
        {
            Debug.Log(action);
            switch (action.ActionCase)
            {
                case Response.ActionOneofCase.AddEntity:
                    AddEntity(action.AddEntity.Entity);
                    break;
                case Response.ActionOneofCase.RemoveEntity:
                    RemoveEntity(action.AddEntity.Entity);
                    break;
                case Response.ActionOneofCase.UpdateEntity:
                    UpdateEntity(action.UpdateEntity.Entity);
                    break;
                case Response.ActionOneofCase.MoveEntity:
                    MoveEntity(action.MoveEntity);
                    break;
                default:
                    break;
            }
        }

        private bool isControlled(string id)
        {
            return _objects.ContainsKey(id) && _objects[id].GetControlType() == ElementType.Owner;
        }

        private void AddEntity(Entity entity)
        {
            if (_objects.ContainsKey(entity.Id)) return;
            var factory = new GameObject().AddComponent<Factory>();
            var script = factory.SpawnElement(entity, _spawnables[entity.Type], new Vector3
            {
                x = entity.Position.X,
                z = entity.Position.Y
            });
            _objects[entity.Id] = script;
        }

        private void RemoveEntity(Entity entity)
        {
            if (isControlled(entity.Id)) return;
            var obj = _objects[entity.Id];
            _objects.Remove(entity.Id);
            obj.Destroy();
        }

        private void UpdateEntity(Entity entity)
        {
            if (isControlled(entity.Id)) return;
            var obj = _objects[entity.Id];
            obj.HandleUpdate(entity);
        }

        private void MoveEntity(MoveEntity moveAction)
        {
            if (isControlled(moveAction.Id)) return;
            _objects[moveAction.Id].HandleUpdate(new Entity
            {
                Position = moveAction.Position,
            });
        }

        private void OnDestroy()
        {
            Disconnect();
        }

        private void OnApplicationQuit()
        {
            Disconnect();
        }

        public async void Disconnect()
        {
            StopAllCoroutines();
            await Task.Delay((int)(1000f / updateFps) + 10);
            GRPC.Disconnect();
        }

        private Position ToPosition(Vector2 position)
        {
            return new Position
            {
                X = position.x,
                Y = position.y
            };
        }
        private Position ToPosition(Vector3 position)
        {
            return new Position
            {
                X = position.x,
                Y = position.z
            };
        }

        private void PostRegistrars()
        {
            foreach (var (id, obj) in _objects)
            {
                var req = new Request
                {
                    AddEntity = new AddEntity
                    {
                        KeepOnDisconnect = !obj.RemoveOnDisconnect(),
                        Entity = new Entity
                        {
                            Id = id,
                            Type = obj.ID(),
                            Name = obj.Name(),
                            Colour = obj.Colour(),
                            Position = ToPosition(obj.GetPosition())
                        }
                    }
                };
                GRPC.SendRequest(req);
            }
        }

            IEnumerator UpdatePosition()
        {
            while (true)
            {
                foreach (var keyValuePair in _objects)
                {
                    if (keyValuePair.Value.GetControlType() == ElementType.Listener) continue;
                    // ideally projectiles should be controlled by the server but i am making them be controlled by the sender for simplicities sake

                    var pos = keyValuePair.Value.GetPosition();
                    if (_objectLastPos.ContainsKey(keyValuePair.Key) &&
                        _objectLastPos[keyValuePair.Key] == pos) continue;
                    _objectLastPos[keyValuePair.Key] = pos;

                    var req = new Request
                    {
                        MoveEntity = new MoveEntity
                        {
                            Id = keyValuePair.Key,
                            Position = ToPosition(pos)
                        }
                    };
                    GRPC.SendRequest(req);
                }

                yield return new WaitForSeconds(1f / updateFps);
            }
        }
    }
}