using UnityEngine;

namespace Online
{
    public interface NetworkedElement
    {
        public Vector2 GetPosition();
        public ElementType GetControlType();
        public void Destroy();
        
        public void HandleUpdate(Entity entity);
    }

    public enum ElementType
    {
        Owner,
        Listener
    }
}