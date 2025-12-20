#!/bin/sh
yq '.data."trigger.on-application-changed"' ./samples/notifications/patch.yaml > ./docs/trigger.yaml
yq '.data."template.app-application-changed"' ./samples/notifications/patch.yaml > ./docs/template.yaml

