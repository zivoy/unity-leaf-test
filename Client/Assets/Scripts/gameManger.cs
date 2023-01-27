using System.Collections;
using System.Collections.Generic;
using Palmmedia.ReportGenerator.Core.Reporting.Builders;
using UnityEngine;

public class gameManger : MonoBehaviour
{
    public GameObject player;
    // Start is called before the first frame update
    void Start()
    {
        NewPlayer(Vector3.zero);
    }
    //todo implement the rest of player connection, make sure that there is a connection
    void NewPlayer(Vector3 pos)
    {
        GameObject playable = Instantiate(player,pos, new Quaternion());
        playable.GetComponent<controller>().Controlled = false;
    }

    // Update is called once per frame
    void Update()
    {
        
    }
}
