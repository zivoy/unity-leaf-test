using UnityEngine;

namespace Online
{
    public interface NetworkedElement
    {
        public Vector2 GetPosition();
        public ElementType GetControlType();
        public string ID();
        
        public void Destroy();
        
        public void HandleUpdate(Entity entity);
    }

    public enum ElementType
    {
        Owner,
        Listener
    }
}