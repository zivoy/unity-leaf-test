using System;
using protoBuff;
using UnityEngine;
using Object = UnityEngine.Object;

namespace Online
{
    public class Factory:MonoBehaviour
    {
        private Entity _factoryEntity;
        private NetworkedElement _factoryObject;
        private Vector3 _factoryPosition;

        public NetworkedElement SpawnElement(Entity entity, GameObject obj, Vector3 position)
        {
            _factoryEntity = entity;
            _factoryPosition = position;
            
            var o = Instantiate(obj, _factoryPosition, new Quaternion());
            _factoryObject = o.GetComponent<NetworkedElement>();
            return _factoryObject;
        }
        
        public void Update()
        {
            if (_factoryObject == null)
            {
                Debug.Log("factory does not have an object");
            }

            try
            {
                _factoryObject.HandleUpdate(_factoryEntity);
            }
            catch (Exception)
            {
                return;
            }
            
            Destroy(gameObject);
        }
    }
}