using System;
using UnityEngine;
using Online;

public class PlayerController : MonoBehaviour, NetworkedElement
{
    public float m_Speed = 5f;

    public bool Controlled = false;

    private Rigidbody _rigidbody;
    // Start is called before the first frame update
    void Start()
    {
        _rigidbody = GetComponent<Rigidbody>();
    }

    // Update is called once per frame
    void Update()
    {
        if (!Controlled) return;
        var input = new Vector3(Input.GetAxis("Horizontal"), 0,Input.GetAxis("Vertical")).normalized;
               
        SetPosition(transform.position + input * (Time.deltaTime * m_Speed));
    }

    public void SetPosition(Vector3 pos)
    {
        _rigidbody.MovePosition(pos);
    }

    public Vector2 GetPosition()
    {
        var pos = transform.position;
        return new Vector2 { x = pos.x, y = pos.y };
    }

    public void Destroy()
    {
        Destroy(gameObject);
    }

    public void HandleUpdate(Entity entity)
    {
        // if (entity.EntityCase != Entity.EntityOneofCase.Player) return;
        var player = entity.Player;
        SetPosition(new Vector3
        {
            x = player.Position.X,
            z = player.Position.Y,
        });
        if (ColorUtility.TryParseHtmlString(player.Colour, out var playerColour))
        {
            GetComponentInChildren<MeshRenderer>().material.color = playerColour;
        }
    }

    public ElementType GetControlType()
    {
        if (Controlled)return ElementType.Owner;
        return ElementType.Listener;
    }
}
