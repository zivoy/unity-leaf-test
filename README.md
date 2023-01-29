# Client centric unity server
**This is an implementation of a terrible server layout that shouldn't be used in actual 
situations where latency actually matters.**

in this client centric approach server just forwards messages and its the client who registered the objects job to send 
updates on it for other users

# setup
run `setup.ps1` or `setup.sh` to install all the required tools and dependencies

requires [go](https://go.dev/) to run/build the server

server code is a heavily modified version of the code on [this blogpost](https://mortenson.coffee/blog/making-multiplayer-game-go-and-grpc/)

## tech
I couldn't get [GRPC for .Net](https://github.com/grpc/grpc-dotnet) 
to work since unity has issues with http/2 
(and i am not paying $70 for [Best HTTP/2](https://assetstore.unity.com/packages/tools/network/best-http-2-155981)), 
so I am using the latest version of the older, soon-to-be deprecated [GRPC Core](https://packages.grpc.io/archive/2022/04/67538122780f8a081c774b66884289335c290cbe-f15a2c1c-582b-4c51-acf2-ab6d711d2c59/index.xml)

<!--
https://openupm.com/packages/ai.transforms.unity-grpc-web/  unity is twat why no http/2

https://github.com/GlitchEnzo/NuGetForUnity
https://github.com/grpc/grpc/tree/master/src/csharp

https://packages.grpc.io/archive/2022/04/67538122780f8a081c774b66884289335c290cbe-f15a2c1c-582b-4c51-acf2-ab6d711d2c59/index.xml ill do it myself
-->