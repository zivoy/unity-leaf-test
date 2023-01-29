using System;
using System.Collections;
using System.Collections.Generic;
using System.Threading.Tasks;
using Grpc.Core;
using UnityEngine;
using protoBuff;
using Request = protoBuff.Request;
//todo add try catches in places
// todo implement list
namespace Online
{
    public class NetworkManager : MonoBehaviour
    {
        private Game.GameClient _client;

        public GameObject[] spawnables;
        private Dictionary<string, GameObject> _spawnables;

        public int updateFps = 60; // update at 60 fps

        private readonly Dictionary<string, NetworkedElement> _objects;
        private readonly Dictionary<string, Vector2> _objectLastPos;
        private AsyncDuplexStreamingCall<Request, Response> _stream;

        private string _token;
        private bool _active;
        private readonly Queue<Request> _queue;

        public NetworkManager()
        {
            _queue = new Queue<Request>();
            _objects = new Dictionary<string, NetworkedElement>();
            _objectLastPos = new Dictionary<string, Vector2>();
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
            _client = new Game.GameClient(Connection.GetInstance().GetChannel());

            Connect();
        }

        /// be careful with this and dont have scripts register on wake since it can lead to recursion 
        public void RegisterObject(GameObject obj, bool removeOnDisconnect = true)
        {
            var id = Guid.NewGuid().ToString();
            var networkedElement = obj.GetComponent<NetworkedElement>();
            var pos = obj.transform.position;
            var req = new Request
            {
                AddEntity = new AddEntity
                {
                    KeepOnDisconnect = !removeOnDisconnect,
                    Entity = new Entity
                    {
                        Id = id,
                        Type = networkedElement.ID(),
                        Name = obj.name,
                        Colour = ColorUtility.ToHtmlStringRGBA(
                            obj.GetComponentInChildren<MeshRenderer>().material.color),
                        Position = new Position
                        {
                            X = pos.x,
                            Y = pos.y
                        }
                    }
                }
            };

            _objects.Add(id, networkedElement);
            SendStreamRequest(req);
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

            // not having it be removed from the dict and destroyed here so it will be done in the broadcast request

            var req = new Request
            {
                RemoveEntity = new RemoveEntity
                {
                    Id = id
                }
            };
            SendStreamRequest(req);
        }

        private void Connect()
        {
            ConnectResponse conn;
            try
            {
                conn = _client.Connect(new ConnectRequest { Session = "The Only One" });
            }
            catch (RpcException e)
            {
                if (e.StatusCode == StatusCode.Unknown) Debug.LogWarning(e.Status.Detail);
                return;
            }

            Debug.Log(conn);
            _token = conn.Token;

            foreach (var entity in conn.Entities)
            {
                AddEntity(entity);
            }

            try
            {
                _stream = _client.Stream(new Metadata
                {
                    new("authorization", _token)
                });
            }
            catch (RpcException e)
            {
                if (e.StatusCode == StatusCode.Unknown) Debug.LogWarning(e.Status.Detail);
                return;
            }

            Task.Run(ReadStreamData);
            StartCoroutine(SendRequests());
            StartCoroutine(UpdatePosition());
            _active = true;
        }

        //todo implement the rest of player connection, make sure that there is a connection / detext disconnect, work out the dispose as well, its not leaving session

        private async void ReadStreamData()
        {
            // while (!await _stream.ResponseStream.MoveNext()) continue;
            try
            {
                while (await _stream.ResponseStream.MoveNext())
                {
                    var action = _stream.ResponseStream.Current;

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
                        case Response.ActionOneofCase.None:
                        default:
                            break;
                    }
                }
            }
            catch (RpcException ex) when (ex.StatusCode == StatusCode.Cancelled)
            {
                Debug.Log("Stream cancelled");
            }
        }

        private void AddEntity(Entity entity)
        {
            if (_objects.ContainsKey(entity.Id)) return;
            var o = Instantiate(_spawnables[entity.Type], new Vector3
            {
                x = entity.Position.X,
                z = entity.Position.Y
            }, new Quaternion());
            var script = o.GetComponent<NetworkedElement>();
            script.HandleUpdate(entity);
            _objects[entity.Id] = script;
        }

        private void RemoveEntity(Entity entity)
        {
            if (!_objects.ContainsKey(entity.Id)) return;
            var obj = _objects[entity.Id];
            _objects.Remove(entity.Id);
            obj.Destroy();
        }

        private void UpdateEntity(Entity entity)
        {
            if (!_objects.ContainsKey(entity.Id)) return;
            var obj = _objects[entity.Id];
            obj.HandleUpdate(entity);
        }
        
        private void OnDestroy()
        {
            Disconnect();
        }

        private void OnApplicationQuit()
        {
            Disconnect();
        }
        
        private async void Disconnect()
        {
            if (!_active) return;
            _active = false;
            StopAllCoroutines();
            Debug.Log("shutting down stream");
            await Task.Delay((int)(1000f / updateFps)+1);
            if (_stream != null)
                await _stream.RequestStream.CompleteAsync();
            Connection.GetInstance().Dispose();
        }

        private void SendStreamRequest(Request req)
        {
            _queue.Enqueue(req);
        }
        
        
        IEnumerator SendRequests()
        {
            while (true)//_queue.Count > 0)
            {
                if (_active && _queue.Count > 0)
                    _stream.RequestStream.WriteAsync(_queue.Dequeue()).Wait();

                yield return null;
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
                        Move = new MoveAction
                        {
                            Id = keyValuePair.Key,
                            Position = new Position
                            {
                                X = pos.x,
                                Y = pos.y
                            }
                        }
                    };
                    _queue.Enqueue(req);
                }

                yield return new WaitForSeconds(1f / updateFps);
            }
        }
    }
}