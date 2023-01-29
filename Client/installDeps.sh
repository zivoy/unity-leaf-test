#!/bin/sh

if [ -d "./Assets/Plugins" ] 
then
    echo "Plugins are already isntalled"
    exit
fi

download (){
  curl -L -o "package.zip" $1
  unzip "package.zip" -d "./Assets/$2" > /dev/null
  rm -f "package.zip"
}

download "https://packages.grpc.io/archive/2022/04/67538122780f8a081c774b66884289335c290cbe-f15a2c1c-582b-4c51-acf2-ab6d711d2c59/csharp/grpc_unity_package.2.47.0-dev202204190851.zip"
echo "installed grpc packages"

download "https://www.nuget.org/api/v2/package/System.Threading.Channels/7.0.0" "Plugins/System.Threading.Channels"
cs Assets/Plugins
mkdir tmp
mv System.Threading.Channels/* tmp/
mv tmp/lib .
rm -rf tmp
cd ../..
echo "installed channels"
