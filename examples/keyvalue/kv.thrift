exception ResourceDoesNotExist {
    1: required string key
    2: optional string message
}

service KeyValue {
    string getValue(1: string key)
        throws (1: ResourceDoesNotExist doesNotExist)
    void setValue(1: string key, 2: string value)
}
