using System.Collections.Generic;
using System.Threading.Tasks;
using Grpc.Core;
using UnityEngine;

public class gameManger : MonoBehaviour
{
    public GameObject player;
    private Game.GameClient _client;
    private Connection _connection;

    public int updateIntervalFps = 60; // update at 60 fps
    private double _lastInterval;

    private Dictionary<string, GameObject> _players;
    private AsyncDuplexStreamingCall<Request, Response> _stream;

    private string _token;
    private string _id;


    // Start is called before the first frame update
    void Start()
    {
        DontDestroyOnLoad(gameObject);
        _connection = Connection.GetInstance();
        _client = new Game.GameClient(_connection.GetChannel());
        _players = new Dictionary<string, GameObject>();

        connect();
    }

    void connect()
    {
        var conn = _client.Connect(new ConnectRequest { Name = "some random ass name" });
        Debug.Log(conn);
        _token = conn.Token;
        _id = conn.Id;

        foreach (var entity in conn.Entities)
        {
            if (entity.Id == conn.Id) continue;
            AddEntity(entity);
        }

        _stream = _client.Stream(new Metadata
        {
            new("authorization", _token)
        });
        Task.Run(ReadStreamData);
    }

    //todo implement the rest of player connection, make sure that there is a connection
    GameObject NewPlayer(Vector3 pos, Color color)
    {
        GameObject playable = Instantiate(player, pos, new Quaternion());
        playable.GetComponent<controller>().Controlled = false;
        playable.GetComponentInChildren<MeshRenderer>().material.color = color;
        return playable;
    }

    // Update is called once per frame
    void Update()
    {
        var timeNow = Time.realtimeSinceStartup;
        var updateInterval = 1f / updateIntervalFps;
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
        if (entity.Id==_id) return;
        if (entity.EntityCase != Entity.EntityOneofCase.Player) return;
        var col = Color.red;
        if (ColorUtility.TryParseHtmlString(entity.Player.Colour, out var playerColour))
        {
            col = playerColour;
        }

        _players[entity.Id] = NewPlayer(new Vector3
        {
            x=entity.Player.Position.X,
            z=entity.Player.Position.Y,
        }, col);
    }

    private void RemoveEntity(Entity entity)
    {
        if (entity.Id==_id) return;
        var obj = _players[entity.Id];
        Destroy(obj);
        _players.Remove(entity.Id);
    }

    private void UpdateEntity(Entity entity)
    {
        if (entity.Id==_id) return;
        if (entity.EntityCase != Entity.EntityOneofCase.Player) return;
        var ent = entity.Player;
        var obj = _players[entity.Id];
        obj.GetComponent<controller>().SetPosition(new Vector3
        {
            x = ent.Position.X,
            z = ent.Position.Y,
        });
        if (ColorUtility.TryParseHtmlString(ent.Colour, out var playerColour))
        {
            obj.GetComponentInChildren<MeshRenderer>().material.color = playerColour;
        }
    }

    private Vector3 _lastPos;
    private async void UpdatePosition()
    {
        var pos = player.transform.position;
        if (_lastPos == pos) return;
        _lastPos = pos;
        var req = new Request
        {
            Move = new Position
            {
                X = pos.x,
                Y = pos.z
            }
        };
        await _stream.RequestStream.WriteAsync(req);
    }

    private void OnDestroy()
    {
        OnApplicationQuit();
    }
    private void OnApplicationQuit()
    {
        Debug.Log("shutting down stream");
        _connection.Dispose();
        _stream.RequestStream.CompleteAsync().Wait();
    }
}