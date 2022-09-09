#!/usr/bin/env bash
echo "Creating mongo role ..."
mongo admin --host localhost -u root -p passwd --eval "db.createRole( { role: 'splunkMonitor', \
                                                      privileges: [ { resource: { db: 'admin', collection: 'system.roles' }, actions: [ 'collStats', 'indexStats' ] }, \
                                                                    { resource: { db: 'admin', collection: 'system.users' }, actions: [ 'collStats', 'indexStats' ] }, \
                                                                    { resource: { db: 'admin', collection: 'system.version' }, actions: [ 'collStats', 'indexStats' ] } ], \
                                                                    roles: [ { role: 'read', db: 'admin' } ] }, { w: 'majority' , wtimeout: 5000 });"
echo "Creating mongo monitor user ..."
mongo admin --host localhost -u root -p passwd --eval "db.createUser( { user: 'test123', pwd: 'test123', \
                                                                        roles: [ { role: 'readAnyDatabase', db: 'admin' }, \
                                                                                 { role: 'clusterMonitor', db: 'admin' }, \
                                                                                 {role: 'splunkMonitor', db: 'admin'} ]});"
echo "Mongo users created."

