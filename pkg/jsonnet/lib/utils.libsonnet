{
    // Returns the final path section of a URL with any data including and 
    // after a dot '.' removed.
    urlToDefaultName(url):: 
        local split = std.split(url, '/');
        local length = std.length(split);
        std.split(split[length-1], '.')[0],

    // Returns if the given value is null or of the given type.
    nullOrIsType(value, type)::
        value == null || std.type(value) == type,

    // Returns if the object has a key and it is the given type.
    notExistsOrType(object, key, type)::
        !std.objectHas(object, key) || std.type(object[key]) == type,
}