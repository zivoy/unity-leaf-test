using System;
using System.Collections.Generic;
using System.Threading.Channels;
using System.Threading.Tasks;
using Google.Protobuf.Collections;
using Grpc.Core;
using protoBuff;
using UnityEngine;
using Channel = System.Threading.Channels.Channel;
using Request = protoBuff.Request;

namespace Online
{
    public delegate void OnMessageCallback(Response action);

    public sealed class GRPC
    {
        public static RepeatedField<Entity> Connect(string session)
        {
            return Grpc()._connect(session);
        }

        public static RepeatedField<protoBuff.Server> List()
        {
            return Grpc()._client.List(new SessionRequest()).Servers;
        }

        public static void StartStream()
        {
            Grpc()._startStream();
        }

        public static void Disconnect()
        {
            Grpc()._disconnect();
        }

        public static void RegisterMessageCallback(OnMessageCallback callback)
        {
            _callback = callback;
        }

        public static async void SendRequest(Request request)
        {
            if (Grpc()._active)
                await Grpc()._queue.Writer.WriteAsync(request);
            else
                Grpc()._idleQueue.Enqueue(request);
        }

        private readonly Game.GameClient _client;
        private AsyncDuplexStreamingCall<Request, Response> _stream;
        private string _token;
        private bool _active;
        private static OnMessageCallback _callback;
        private readonly Channel<Request, Request> _queue;
        private readonly Queue<Request> _idleQueue;

        private GRPC()
        {
            _queue = Channel.CreateUnbounded<Request>();
            _idleQueue = new Queue<Request>();
            _client = new Game.GameClient(Connection.GetInstance().GetChannel());
        }

        private static GRPC _instance;

        private static GRPC Grpc()
        {
            _instance ??= new GRPC();
            return _instance;
        }

        private RepeatedField<Entity> _connect(string session)
        {
            var conn = _client.Connect(new ConnectRequest { Session = session });
            _token = conn.Token;
            return conn.Entities;
        }

        private void _startStream()
        {
            if (_client == null)
            {
                throw new Exception("No connection");
            }

            _stream = _client.Stream(new Metadata
            {
                new("authorization", _token)
            });
            _active = true;
            Task.Run(_readStreamData);
            Task.Run(_messageWriter);

            while (_idleQueue.Count > 0)
                SendRequest(_idleQueue.Dequeue());
        }

        private async void _disconnect()
        {
            if (!_active) return;
            _active = false;
            await _queue.Writer.WriteAsync(new Request());
            Debug.Log("Shutting down stream");
            Connection.GetInstance().Dispose();
            _instance = null;
        }

        private async void _readStreamData()
        {
            try
            {
                while (await _stream.ResponseStream.MoveNext())
                {
                    var action = _stream.ResponseStream.Current;
                    _callback(action);
                }
            }
            catch (RpcException ex) when (ex.StatusCode == StatusCode.Cancelled)
            {
                Debug.Log("Stream cancelled");
            }
        }

        private async void _messageWriter()
        {
            while (true)
            {
                var req = await _queue.Reader.ReadAsync();
                if (!_active)
                {
                    await _stream.RequestStream.CompleteAsync();
                    return;
                }

                await _stream.RequestStream.WriteAsync(req);
            }
        }
    }
}