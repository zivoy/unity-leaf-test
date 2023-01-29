using UnityEngine;
using protoBuff;

namespace Online
{
    public interface NetworkedElement
    {
        public Vector2 GetPosition();
        public ElementType GetControlType();
        public string ID();
        public string Name();
        public string Colour();
        public bool RemoveOnDisconnect();
        
        public void Destroy();
        
        public void HandleUpdate(Entity entity);
    }

    public enum ElementType
    {
        Owner,
        Listener
    }
}