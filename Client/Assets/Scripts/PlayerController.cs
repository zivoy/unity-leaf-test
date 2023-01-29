using Google.Protobuf;
using UnityEngine;
using Online;
using protoBuff;

public class PlayerController : MonoBehaviour, NetworkedElement
{
    public float m_Speed = 5f;

    public bool Controlled = false;

    private Rigidbody _rigidbody;

    // Start is called before the first frame update
    void Start()
    {
        _rigidbody = GetComponent<Rigidbody>();

        if (Controlled)
        {
            GetComponentInChildren<MeshRenderer>().material.color = Random.ColorHSV();

            var networkManager = FindObjectOfType<NetworkManager>();
            networkManager.RegisterObject(this);
        }
    }

    // Update is called once per frame
    void Update()
    {
        if (!Controlled) return;
        var input = new Vector3(Input.GetAxis("Horizontal"), 0, Input.GetAxis("Vertical")).normalized;

        setPosition(transform.position + input * (Time.deltaTime * m_Speed));
    }

    public string ID()
    {
        return "PLAYER";
    }

    private void setPosition(Vector3 pos)
    {
        _rigidbody.MovePosition(pos);
    }

    public Vector2 GetPosition()
    {
        var pos = transform.position;
        return new Vector2 { x = pos.x, y = pos.z };
    }

    public void Destroy()
    {
        Destroy(gameObject);
    }

    public void HandleUpdate(Entity entity)
    {
        setPosition(new Vector3
        {
            x = entity.Position.X,
            z = entity.Position.Y,
        });

        if (entity.Data == "") return;
        GetComponentInChildren<MeshRenderer>().material.color = getColour(entity.Data);
    }

    public ElementType GetControlType()
    {
        return Controlled ? ElementType.Owner : ElementType.Listener;
    }

    public bool RemoveOnDisconnect()
    {
        return true;
    }

    public string Data()
    {
        var objName = gameObject.name;
        var colour = ColorUtility.ToHtmlStringRGBA(
            GetComponentInChildren<MeshRenderer>().material.color);
        colour = colour.Substring(0, 6);
        return colour + objName;
    }

    private Color getColour(string dataString)
    {
        var colour = "#" + dataString.Substring(0, 6);
        var col = Color.red;
        if (ColorUtility.TryParseHtmlString(colour, out var playerColour))
        {
            col = playerColour;
        }

        return col;
    }
}