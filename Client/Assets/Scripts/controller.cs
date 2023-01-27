using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

public class controller : MonoBehaviour
{
    public float m_Speed = 5f;

    [NonSerialized] public bool Controlled = true;

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
        Vector3 input = new Vector3(Input.GetAxis("Horizontal"), 0, Input.GetAxis("Vertical")).normalized;
               
        SetPosition(transform.position + input * (Time.deltaTime * m_Speed));
    }

    public void SetPosition(Vector3 pos)
    {
        _rigidbody.MovePosition(pos);
    }
}
