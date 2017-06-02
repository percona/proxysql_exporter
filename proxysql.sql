INSERT INTO mysql_servers(hostgroup_id, hostname, port) VALUES (1, 'mysql', 3306);
INSERT INTO mysql_servers(hostgroup_id, hostname, port) VALUES (1, 'percona-server', 3306);
LOAD MYSQL SERVERS TO RUNTIME;
SAVE MYSQL SERVERS TO DISK;

INSERT INTO mysql_users(username, password, default_hostgroup) VALUES ('root', '', 1);
INSERT INTO mysql_users(username, password, default_hostgroup) VALUES ('monitor', 'monitor', 1);
LOAD MYSQL USERS TO RUNTIME;
SAVE MYSQL USERS TO DISK;

SELECT * FROM mysql_servers;
SELECT * FROM mysql_users;
