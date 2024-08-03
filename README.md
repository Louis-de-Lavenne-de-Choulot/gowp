# gowp
 
This is a Program inspired from Wordpress. The goal is to have a basic server with a plugin system.
The program will have one mandatory plugin for login.
All other plugins will be optional.

## Expectations
- The program will be written in go.
- The program will up a server
- The program will have a plugin system
- The program will have a login system through a plugin
- The program will have an admin panel

## Basic Server
The server should be able to :
- Listen to a port
- Init DB Connection
- Init All Routes (and plugins ?)
- Serve Template with informations from map[string]interface{}
- Call middlewares
    - Call plugins functions
        - Pass data pointer *map[string]interface{} to plugin
- Call plugins routes


## Plugins

Plugins should be able to :
- Add public routes
- Specify Role Levels to access the plugin's routes
- Add middlewares (/admin/login_processing)
- Add Global midddlewares (*)
- Return Error
- Contact Database
- Change Informations before templates are rendered




name : nom du plugin
base_route : route de base après /
version : version du plugin
author : auteur du plugin
description : description du plugin
languages : an array of languages supported by the plugin