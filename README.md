# appServer

[![Build Status](https://travis-ci.org/boxproject/appServerGo.svg?branch=master)](https://travis-ci.org/boxproject/appServerGo) [![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)](https://www.apache.org/licenses/LICENSE-2.0) [![language](https://img.shields.io/badge/golang-%5E1.10-blue.svg)]()

The Staff-Manager App Server for Enterprise Token Safe BOX
## Update 
This project has been migrated to a new repository. Please chekc update from [apiServer](https://github.com/boxproject/apiServer) for V1.0 and subsequent versions.

## Before Use

- Modify the configuration file `config.toml.example` to improve proxy server information and your MySQL configuration information.
- Rewrite the file name  `config.toml.example` to `config.toml`.
- Init your MySQL with the file `/db/box.sql`.
- It would be best to modify the server mode from `debug` to `release`.

## Quickstart

### Get source code

~~~
$ git clone git@gitlab.2se.com:boxproject/appServerGo.git
~~~

## Build

~~~
$ cd appServerGo && go build
~~~

## Start

~~~
$ ./appServerGo start
~~~


## Stop

~~~
$ ./appServerGo stop
~~~

## Documentation

To check out API docs, visit `/doc/API.md`

## Licence

Licensed under the Apache License, Version 2.0, Copyright 2018. box.la authors.

~~~
 Copyright 2018. box.la authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
~~~
