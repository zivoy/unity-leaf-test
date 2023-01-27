using UnityEngine;

public class ColourClient
{
    private readonly ColourGenerator.ColourGeneratorClient _client;
    private readonly Connection _connection;

    internal ColourClient()
    {

        _connection = Connection.GetInstance();
        _client = new ColourGenerator.ColourGeneratorClient(_connection.GetChannel());
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
        _connection.Dispose();
    }
}
