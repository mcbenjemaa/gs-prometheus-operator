load('ext://restart_process', 'docker_build_with_restart')

VERSION="v1alpha1"
KIND="Promethues"
IMG="medchiheb/gs-prometheus-operator:v0.1.0-alpha1"

DOCKERFILE = '''FROM golang:alpine
WORKDIR /
COPY ./build/manager /
USER 65532:65532 
CMD ["/manager"]
'''

def yaml():
    return local('cd config/manager; kustomize edit set image controller=' + IMG + '; cd ../..; kustomize build config/default')

def manifests():
    return 'make manifests'

def generate():
    return 'make generate'

def vetfmt():
    return 'go vet ./...; go fmt ./...'

def binary():
    return 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o build/manager main.go'


DIRNAME = os.path.basename(os. getcwd())


local(manifests()) 
local(generate())

local_resource('CRD', 'make install', deps=["api"])

k8s_yaml(yaml())

deps = ['controllers', 'main.go']
deps.append('api')

local_resource('Watch&Compile', binary(), deps=deps, ignore=['*/*/zz_generated.deepcopy.go'])

local_resource('Sample YAML', 'make sample', deps=["./config/samples"], resource_deps=[DIRNAME + "-controller-manager"])

docker_build_with_restart(IMG, '.', 
    dockerfile_contents=DOCKERFILE,
    entrypoint='/manager',
    only=['./build/manager'],
    live_update=[
        sync('./build/manager', '/manager'),
    ]) 