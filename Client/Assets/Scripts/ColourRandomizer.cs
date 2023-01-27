using System;
using UnityEngine;

public class ColourRandomizer : MonoBehaviour
{
    public MeshRenderer player;

    private readonly Color _defaultColor = Color.red;
    private ColourClient _colourClient;

    private void Start()
    {
        _colourClient = new ColourClient();
    }

    private Color GetColour(Color currentColor)
    {
        var currentColourString = ColorUtility.ToHtmlStringRGBA(currentColor);
        var newColourString = _colourClient.GetRandomColour(currentColourString);

        if (ColorUtility.TryParseHtmlString(newColourString, out var newColour))
        {
            return newColour;
        }

        Debug.Log("Error parsing the colour string: " + newColourString);
        Debug.Log("Setting to default colour: " + _defaultColor);
        return _defaultColor;
    }

    public void ChangeColour()
    {
        player.material.color = GetColour(player.material.color);
    }
}