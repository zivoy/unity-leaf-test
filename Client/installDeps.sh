#!/bin/sh

if [ -d "./Assets/Plugins" ] 
then
    echo "Plugins are already isntalled"
    exit
fi

LIB_URL="https://packages.grpc.io/archive/2022/04/67538122780f8a081c774b66884289335c290cbe-f15a2c1c-582b-4c51-acf2-ab6d711d2c59/csharp/grpc_unity_package.2.47.0-dev202204190851.zip"
curl -L -o "unity-package.zip" $LIB_URL
unzip "unity-package.zip" -d "./Assets/" > /dev/null
rm -f "unity-package.zip"