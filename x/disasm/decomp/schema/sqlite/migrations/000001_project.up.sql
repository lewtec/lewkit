CREATE TABLE artifact (
    name TEXT PRIMARY KEY,
    format TEXT NOT NULL,
    architecture INTEGER NOT NULL,
    mode INTEGER NOT NULL,
    entry INTEGER NOT NULL,
    image BLOB NOT NULL,
    path TEXT NOT NULL DEFAULT '',
    digest TEXT NOT NULL DEFAULT ''
);

CREATE TABLE section (
    artifact TEXT NOT NULL REFERENCES artifact(name) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    name TEXT NOT NULL,
    address INTEGER NOT NULL,
    size INTEGER NOT NULL,
    executable INTEGER NOT NULL,
    writable INTEGER NOT NULL,
    readable INTEGER NOT NULL,
    PRIMARY KEY (artifact, position)
);

CREATE TABLE symbol (
    artifact TEXT NOT NULL REFERENCES artifact(name) ON DELETE CASCADE,
    address INTEGER NOT NULL,
    name TEXT NOT NULL,
    PRIMARY KEY (artifact, address)
);

CREATE TABLE function (
    artifact TEXT NOT NULL REFERENCES artifact(name) ON DELETE CASCADE,
    address INTEGER NOT NULL,
    name TEXT NOT NULL,
    signature TEXT NOT NULL,
    comment TEXT NOT NULL,
    source TEXT NOT NULL,
    size INTEGER NOT NULL,
    PRIMARY KEY (artifact, address)
);

CREATE TABLE instruction (
    artifact TEXT NOT NULL,
    function_address INTEGER NOT NULL,
    position INTEGER NOT NULL,
    address INTEGER NOT NULL,
    size INTEGER NOT NULL,
    bytes BLOB NOT NULL,
    mnemonic TEXT NOT NULL,
    operands TEXT NOT NULL,
    instruction_id INTEGER NOT NULL,
    PRIMARY KEY (artifact, function_address, position),
    FOREIGN KEY (artifact, function_address) REFERENCES function(artifact, address) ON DELETE CASCADE
);

CREATE TABLE block (
    artifact TEXT NOT NULL,
    function_address INTEGER NOT NULL,
    position INTEGER NOT NULL,
    address INTEGER NOT NULL,
    kind TEXT NOT NULL,
    target INTEGER NOT NULL,
    fall INTEGER NOT NULL,
    instruction_count INTEGER NOT NULL,
    PRIMARY KEY (artifact, function_address, position),
    FOREIGN KEY (artifact, function_address) REFERENCES function(artifact, address) ON DELETE CASCADE
);

CREATE TABLE reference (
    artifact TEXT NOT NULL,
    function_address INTEGER NOT NULL,
    position INTEGER NOT NULL,
    from_address INTEGER NOT NULL,
    to_address INTEGER NOT NULL,
    kind TEXT NOT NULL,
    PRIMARY KEY (artifact, function_address, position),
    FOREIGN KEY (artifact, function_address) REFERENCES function(artifact, address) ON DELETE CASCADE
);

CREATE TABLE function_string (
    artifact TEXT NOT NULL,
    function_address INTEGER NOT NULL,
    position INTEGER NOT NULL,
    text TEXT NOT NULL,
    PRIMARY KEY (artifact, function_address, position),
    FOREIGN KEY (artifact, function_address) REFERENCES function(artifact, address) ON DELETE CASCADE
);

CREATE TABLE note (
    artifact TEXT NOT NULL REFERENCES artifact(name) ON DELETE CASCADE,
    address INTEGER NOT NULL,
    text TEXT NOT NULL,
    PRIMARY KEY (artifact, address)
);

CREATE TABLE image_string (
    artifact TEXT NOT NULL REFERENCES artifact(name) ON DELETE CASCADE,
    address INTEGER NOT NULL,
    text TEXT NOT NULL,
    PRIMARY KEY (artifact, address)
);
