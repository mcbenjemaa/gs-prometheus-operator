# DEV


#### Tilt

```
ctlptl create registry ctlptl-registry --port=5005
ctlptl create cluster kind --registry=ctlptl-registry
```

#### Build

```
make docker-build docker-push IMG=medchiheb/gs-prometheus-operator:v0.1.0-alpha-1

make deploy IMG=medchiheb/gs-prometheus-operator:v0.1.0-alpha-1
```