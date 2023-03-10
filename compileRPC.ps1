$OperatingSystem = "windows"
$Version = "x64"
$OS = $OperatingSystem +"_"+$Version
$GRPC_PATH ="./Grpc-Tools"
$GRPC_URL = "https://www.nuget.org/api/v2/package/Grpc.Tools/"
if (-not(Test-Path -Path $GRPC_PATH)) {
    $temp_dir = $GRPC_PATH+"/tmp" 
    mkdir $temp_dir -ea 0 > $null
    cd $temp_dir
    Invoke-WebRequest -URI $GRPC_URL -OutFile "tmp.zip"
    Expand-Archive "tmp.zip" -DestinationPath "."

    cd ..
    Get-ChildItem -Path "tmp/tools/$OS" -Recurse | Move-Item -Destination "."
    rm "tmp" -r -force
    cd ..
}

$protoc = $GRPC_PATH+"/protoc.exe"

if (Get-Command "go" -errorAction SilentlyContinue){
    if (-not(Get-Command "protoc-gen-go" -errorAction SilentlyContinue)){
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    }
    if (-not(Get-Command "protoc-gen-go-grpc" -errorAction SilentlyContinue)){
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    }
    & $protoc --go-grpc_out=Server/ --go_out=Server/ grpc/*.proto
    echo "built go protos"
}

$csharpPlugin = $GRPC_PATH+"/grpc_csharp_plugin.exe"
& $protoc --csharp_out=Client/Assets/Scripts/Online/Generated --grpc_out=Client/Assets/Scripts/Online/Generated --plugin=protoc-gen-grpc=$csharpPlugin grpc/*.proto
echo "built unity protos"
