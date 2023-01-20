--------------------------------------------------------------------------------
-- Up
--------------------------------------------------------------------------------

CREATE TABLE Chain (
  id    INTEGER      PRIMARY KEY,
  name  VARCHAR(30)  NOT NULL UNIQUE
);

INSERT INTO Chain (name)
VALUES
  ('localhost'),
  ('baobab'),
  ('cypress');

CREATE TABLE VrfKey (
  id      INTEGER        PRIMARY KEY,
  chainId INTEGER        NOT NULL,
  sk      CHARACTER(64)  NOT NULL,
  pk      CHARACTER(130) NOT NULL,
  pk_x    CHARACTER(77)  NOT NULL,
  pk_y    CHARACTER(77)  NOT NULL,
  CONSTRAINT VrfKey_fk_chainId FOREIGN KEY (chainId)
    REFERENCES Chain (id) ON UPDATE CASCADE ON DELETE CASCADE
);

INSERT INTO VrfKey (chainId, sk, pk, pk_x, pk_y)
VALUES (
  (SELECT id from Chain WHERE name = 'localhost'),
  'a0282885368c7f3046749babc93724c25e48c95fe790d625a2cedef0f194a73f',
  '04d26433ce8f3cd46a98d2d24ee3c4e02688f5b73f61a489df611b06a59e023a11756cfe3662aba23a471f836da3b171333425213cc9e3d35ab0f2ae4247ac8c8f',
  '95162740466861161360090244754314042169116280320223422208903791243647772670481',
  '53113177277038648369733569993581365384831203706597936686768754351087979105423'
);

CREATE TABLE Service (
  id    INTEGER      PRIMARY KEY,
  name  VARCHAR(30)  NOT NULL UNIQUE
);

INSERT INTO Service (name)
VALUES
  ('VRF'),
  ('Aggregator'),
  ('RequestResponse');


CREATE TABLE Listener (
  id         INTEGER        PRIMARY KEY,
  chainId    INTEGER        NOT NULL,
  serviceId  INTEGER        NOT NULL,
  address    CHARACTER(42)  NOT NULL,
  eventName  VARCHAR(255)   NOT NULL,
  CONSTRAINT Listener_fk_chainId FOREIGN KEY (chainId)
    REFERENCES Chain (id) ON UPDATE CASCADE ON DELETE CASCADE
  CONSTRAINT Listener_fk_serviceId FOREIGN KEY (serviceId)
    REFERENCES Service (id) ON UPDATE CASCADE ON DELETE CASCADE
);

INSERT INTO Listener (chainId, serviceId, address, eventName)
VALUES
  ((SELECT id from Chain WHERE name = 'localhost'),
  (SELECT id from Service WHERE name = 'VRF'),
           '0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0', 'RandomWordsRequested'),
  ((SELECT id from Chain WHERE name = 'localhost'),
  (SELECT id from Service WHERE name = 'Aggregator'),
           '0xa513E6E4b8f2a923D98304ec87F64353C4D5C853', 'NewRound'),
  ((SELECT id from Chain WHERE name = 'localhost'),
  (SELECT id from Service WHERE name = 'RequestResponse'),
           '0x45778c29A34bA00427620b937733490363839d8C', 'Requested');

CREATE TABLE Kv (
  id       INTEGER       PRIMARY KEY,
  chainId  INTEGER       NOT NULL,
  key      VARCHAR(255)  NOT NULL,
  value    VARCHAR(255)  NOT NULL
);

INSERT INTO Kv (chainId, key, value)
VALUES
  ((SELECT id from Chain WHERE name = 'localhost'), 'PROVIDER', 'http://127.0.0.1:8545'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'REDIS_HOST', 'localhost'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'REDIS_PORT', '6379'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'PUBLIC_KEY', '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'PRIVATE_KEY', '0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'LOCAL_AGGREGATOR', 'median');

--------------------------------------------------------------------------------
-- Down
--------------------------------------------------------------------------------

DROP TABLE Chain;
DROP TABLE VrfKey;
DROP TABLE Service;
DROP TABLE Listener;
DROP TABLE Kv;
