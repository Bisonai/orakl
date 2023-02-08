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
           '0x0165878a594ca255338adfa4d48449f69242eb8f', 'RandomWordsRequested'),
  ((SELECT id from Chain WHERE name = 'localhost'),
  (SELECT id from Service WHERE name = 'Aggregator'),
           '0xa513E6E4b8f2a923D98304ec87F64353C4D5C853', 'NewRound'),
  ((SELECT id from Chain WHERE name = 'localhost'),
  (SELECT id from Service WHERE name = 'RequestResponse'),
           '0xe7f1725e7734ce288f8367e1bb143e90bb3f0512', 'DataRequested');

CREATE TABLE Kv (
  id       INTEGER       PRIMARY KEY,
  chainId  INTEGER       NOT NULL,
  key      VARCHAR(255)  NOT NULL,
  value    VARCHAR(255)  NOT NULL,
  UNIQUE(chainId, key) ON CONFLICT FAIL
);

INSERT INTO Kv (chainId, key, value)
VALUES
  ((SELECT id from Chain WHERE name = 'localhost'), 'PROVIDER_URL', 'http://127.0.0.1:8545'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'REDIS_HOST', 'localhost'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'REDIS_PORT', '6379'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'PRIVATE_KEY', '0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'PUBLIC_KEY', '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'LOCAL_AGGREGATOR', 'MEDIAN'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'HEALTH_CHECK_PORT', '8888'),
  ((SELECT id from Chain WHERE name = 'localhost'), 'SLACK_WEBHOOK_URL', ''),        
  ((SELECT id from Chain WHERE name = 'localhost'), 'LISTENER_DELAY', '500');

CREATE TABLE Adapter (
  id         INTEGER   PRIMARY KEY,
  adapterId  CHAR(66)  NOT NULL UNIQUE,
  chainId    INTEGER   NOT NULL,
  data       TEXT      NOT NULL,
  CONSTRAINT Adapter_fk_chainId FOREIGN KEY (chainId)
    REFERENCES Chain (id) ON UPDATE CASCADE ON DELETE CASCADE
);

INSERT INTO Adapter (chainId, adapterId, data)
VALUES
  ((SELECT id from Chain WHERE name = 'localhost'),
  '0xc9f7c0b3a3e75ca24b9d84ab2ebbcad5cff09317f87532e90b79bf2ebbb327a3',
  '{
    "id": "0xc9f7c0b3a3e75ca24b9d84ab2ebbcad5cff09317f87532e90b79bf2ebbb327a3",
    "active": false,
    "name": "ETH/USD",
    "jobType": "DATA_FEED",
    "decimals": "8",
    "feeds": [
        {
            "url": "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=ETH&tsyms=USD",
            "headers": {
                "Content-Type": "application/json"
            },
            "method": "GET",
            "reducers": [
                {
                    "function": "PARSE",
                    "args": [
                        "RAW",
                        "ETH",
                        "USD",
                        "PRICE"
                    ]
                },
                {
                    "function": "POW10",
                    "args": "8"
                },
                {
                    "function": "ROUND"
                }
            ]
        }
    ]
  }'),
  ((SELECT id from Chain WHERE name = 'localhost'),
  '0x00d5130063bee77302b133b5c6a0d6aede467a599d251aec842d24abeb5866a5',
  '{
    "id": "0x00d5130063bee77302b133b5c6a0d6aede467a599d251aec842d24abeb5866a5",
    "active": true,
    "name": "KLAY/USD",
    "jobType": "DATA_FEED",
    "decimals": "8",
    "feeds": [
        {
            "url": "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD",
            "headers": {
                "Content-Type": "application/json"
            },
            "method": "GET",
            "reducers": [
                {
                    "function": "PARSE",
                    "args": ["RAW", "KLAY", "USD", "PRICE"]
                },
                {
                    "function": "POW10",
                    "args": "8"
                },
                {
                    "function": "ROUND"
                }
            ]
        },
        {
            "url": "https://api.coingecko.com/api/v3/simple/price?ids=klay-token&vs_currencies=usd",
            "headers": {
                "Content-Type": "application/json"
            },
            "method": "GET",
            "reducers": [
                {
                    "function": "PARSE",
                    "args": ["klay-token", "usd"]
                },
                {
                    "function": "POW10",
                    "args": "8"
                },
                {
                    "function": "ROUND"
                }
            ]
        },
        {
            "url": "https://api.coinbase.com/v2/exchange-rates?currency=KLAY",
            "headers": {
                "Content-Type": "application/json"
            },
            "method": "GET",
            "reducers": [
                {
                    "function": "PARSE",
                    "args": ["data", "rates", "USD"]
                },
                {
                    "function": "POW10",
                    "args": "8"
                },
                {
                    "function": "ROUND"
                }
            ]
        }
    ]
  }');

CREATE TABLE Aggregator (
  id            INTEGER   PRIMARY KEY,
  aggregatorId  CHAR(66)  NOT NULL,
  chainId       INTEGER   NOT NULL,
  adapterId     INTEGER   NOT NULL,
  data          TEXT      NOT NULL,
  UNIQUE(aggregatorId, chainId) ON CONFLICT FAIL,
  CONSTRAINT Aggregator_fk_chainId FOREIGN KEY (chainId)
    REFERENCES Chain (id) ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT Aggregator_fk_adapterId FOREIGN KEY (adapterId)
    REFERENCES Adapter (id) ON UPDATE CASCADE ON DELETE CASCADE
);

INSERT INTO Aggregator (chainId, adapterId, aggregatorId, data)
VALUES
  ((SELECT id from Chain WHERE name = 'localhost'),
   (SELECT id from Adapter WHERE json_extract(Adapter.data, '$.id')='0xc9f7c0b3a3e75ca24b9d84ab2ebbcad5cff09317f87532e90b79bf2ebbb327a3'),
   '0x4bbb04ac1bd973770a0b8e585a41147648980f3094ee7ac5597b2a987e9e96a9',
  '{
    "id": "0x4bbb04ac1bd973770a0b8e585a41147648980f3094ee7ac5597b2a987e9e96a9",
    "address": "0x0000000000000000000000000000000000000000",
    "active": false,
    "name": "ETH/USD",
    "fixedHeartbeatRate": {
        "active": true,
        "value": 10000
    },
    "randomHeartbeatRate": {
        "active": false,
        "value": 9000
    },
    "threshold": 0.09,
    "absoluteThreshold": 0.01,
    "adapterId": "0xc9f7c0b3a3e75ca24b9d84ab2ebbcad5cff09317f87532e90b79bf2ebbb327a3"
  }'),
  ((SELECT id from Chain WHERE name = 'localhost'),
   (SELECT id from Adapter WHERE json_extract(Adapter.data, '$.id')='0x00d5130063bee77302b133b5c6a0d6aede467a599d251aec842d24abeb5866a5'),
   '0x2d5d94df99ccad54f0f6a9d38f2340db793833947f86b207dcda38583dd263fa',
  '{
    "id": "0x2d5d94df99ccad54f0f6a9d38f2340db793833947f86b207dcda38583dd263fa",
    "address": "0x2279B7A0a67DB372996a5FaB50D91eAA73d2eBe6",
    "active": true,
    "name": "KLAY/USD",
    "fixedHeartbeatRate": {
        "active" : true,
        "value": 15000
    },
    "randomHeartbeatRate": {
        "active": false,
        "value": 2000
    },
    "threshold": 0.05,
    "absoluteThreshold": 0.1,
    "adapterId": "0x00d5130063bee77302b133b5c6a0d6aede467a599d251aec842d24abeb5866a5"
  }');

--------------------------------------------------------------------------------
-- Down
--------------------------------------------------------------------------------

DROP TABLE Chain;
DROP TABLE VrfKey;
DROP TABLE Service;
DROP TABLE Listener;
DROP TABLE Kv;
DROP TABLE Adapter;
DROP TABLE Aggregator;
