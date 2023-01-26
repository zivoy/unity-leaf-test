using System;
using Grpc.Core;
using UnityEngine;

public class ColourClient
{
    private ColourGenerator.ColourGeneratorClient _client;
    private readonly Channel _channel;
    private readonly string _serverAddress = "localhost:50051";

    internal ColourClient()
    {

        _channel = new Channel(_serverAddress, ChannelCredentials.Insecure);
        _client = new ColourGenerator.ColourGeneratorClient(_channel);
    }

    internal string GetRandomColour(string currentColour)
    {
        var randomColour = _client.GetRandColour(new CurrentColour { Colour = currentColour });
        Debug.Log("Client is currently using colour: " + currentColour +
                  "switching to: " + randomColour.Colour);
        return randomColour.Colour;
    }

    private void OnDisable()
    {
        Debug.Log("Shutting down channel");
        _channel.ShutdownAsync().Wait();
    }
}
