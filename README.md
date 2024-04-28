# TAPR
[![](https://github.com/Above-Os/tapr/actions/workflows/build.yaml/badge.svg?branch=main)](https://github.com/Above-Os/tapr/actions/workflows/build.yaml)

Terminus Application Runtime

## Image uploader
Provides an upload gateway to upload an image into user space (user's Home dir)

## secret vault
Provides a gateway to create / get / list / delete a secret key to the user's vault - infisical

## sys event
Fire the events where the system has something happened like `a new user created` `an app installed` `CPU high load` etc.

## upload
A general upload sidecar is provided for apps to upload files into the cluster

## ws-gateway
A general WebSocket sidecar is provided for apps to create a WebSocket channel easily

## middleware-operator
Provides some cluster-hosted middleware like RDB, NoSQL, Cache, and Search Index, as apps' persistence strategies in Terminus.
