using System;
using System.Collections.Generic;
using System.Threading.Tasks;
using Grpc.Core;
using UnityEngine;

namespace Online
{
    public class NetworkManager : MonoBehaviour
    {
        private Game.GameClient _client;

        public GameObject[] Spawnables;
        private Dictionary<string, GameObject> _spawnables;

        //todo make server handle register and unregister events

        public int updateFps = 60; // update at 60 fps
        private double _lastInterval;

        private Dictionary<string, NetworkedElement> _objects;
        private Dictionary<string, Vector2> _objectLastPos;
        private AsyncDuplexStreamingCall<Request, Response> _stream;

        private string _token;
        private bool _active;


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
            foreach (var spawnable in Spawnables)
            {
                if (_spawnables.ContainsKey(spawnable.name))
                    throw new Exception("name collision with " + spawnable.name);
                _spawnables[spawnable.name]=spawnable;
                
                if (spawnable.GetComponent<NetworkedElement>() != null) continue;
                // Destroy(gameObject);
                throw new Exception(spawnable.name + " is missing an script that implements NetworkedElement");
            }

            DontDestroyOnLoad(gameObject);
            _client = new Game.GameClient(Connection.GetInstance().GetChannel());
            _objects = new Dictionary<string, NetworkedElement>();

            Connect();
        }

        public void RegisterObject(NetworkedElement obj)
        {
            throw new NotImplementedException("make me");
        }
        public void UnregisterObject(NetworkedElement obj)
        {
            throw new NotImplementedException("make me");
        }

        private void Connect()
        {
            var conn = _client.Connect(new ConnectRequest { Name = "some random ass name" });
            Debug.Log(conn);
            _token = conn.Token;

            foreach (var entity in conn.Entities)
            {
                AddEntity(entity);
            }

            _stream = _client.Stream(new Metadata
            {
                new("authorization", _token)
            });
            Task.Run(ReadStreamData);
            _active = true;
        }

        //todo implement the rest of player connection, make sure that there is a connection

        // Update is called once per frame
        public void Update()
        {
            if (!_active) return;
            var timeNow = Time.realtimeSinceStartup;
            var updateInterval = 1f / updateFps;
            if (timeNow < _lastInterval + updateInterval)
            {
                return;
            }

            _lastInterval = timeNow;
            UpdatePosition();
        }

        private async void ReadStreamData()
        {
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
            var o = Instantiate(_spawnables[entity.Type]);
            o.GetComponent<NetworkedElement>().HandleUpdate(entity);
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

        private async void UpdatePosition()
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
                    Move = new Position
                    {
                        X = pos.x,
                        Y = pos.y
                    }
                };
                await _stream.RequestStream.WriteAsync(req);
            }
        }

        private void OnDestroy()
        {
            Disconnect();
        }

        private void OnApplicationQuit()
        {
            Disconnect();
        }

        private void Disconnect()
        {
            if (!_active) return;
            Debug.Log("shutting down stream");
            Connection.GetInstance().Dispose();
            _stream?.RequestStream.CompleteAsync().Wait();
            _active = false;
        }
    }
}