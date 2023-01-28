using Grpc.Core;
using UnityEngine;

namespace Online
{
    public sealed class Connection
    {
        private Channel _channel;
        private string _address = "localhost:50051";

        private Connection()
        {
        }

        private static Connection _instance;

        public static Connection GetInstance()
        {
            _instance ??= new Connection();

            return _instance;
        }

        public Channel GetChannel()
        {
            if (_channel != null && _channel.State != ChannelState.Shutdown)
            {
                return _channel;
            }

            _channel = new Channel(_address, ChannelCredentials.Insecure);
            return _channel;
        }

        public async void Dispose()
        {
            Debug.Log("Shutting down channel");
            if (_channel != null)
                await _channel.ShutdownAsync();
            _instance = null;
        }

        public Channel ChangeAddress(string address)
        {
            Dispose();
            _address = address;
            return GetChannel();
        }
    }
}