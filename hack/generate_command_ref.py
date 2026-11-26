#!/usr/bin/env python3

import sys
import os
import re
import textwrap

# A flag of the help output: an optional short name, the long name, an optional
# type and the description, which is separated from the rest by two spaces or
# more.
FLAG_RE = re.compile("^\\s*(?:(-\\w,)\\s+)?(--[A-Za-z0-9-]+)(?:\\s(.*))?$")

# The type names pflag prints for a flag. A first word of the description that
# is none of them means that the flag has no type, as a boolean flag has.
FLAG_TYPES = {
    "bool", "boolSlice", "bytesBase64", "bytesHex", "count", "duration",
    "durationSlice", "float", "float32", "float64", "int", "int8", "int16", "int32",
    "int64", "intSlice", "ip", "ipMask", "ipNet", "ipSlice", "string",
    "stringArray", "stringSlice", "stringToInt", "stringToString", "strings",
    "uint", "uint8", "uint16", "uint32", "uint64", "uintSlice",
}


def printline(formatstr, shortOpt, longOpt, argType, envVar, descr):
    #envVar = envVar.replace("_", "\_")
    descrLines = textwrap.wrap(descr, width=60)
    opts = longOpt
    if shortOpt != "":
        opts = opts + " " + shortOpt
    if len(descrLines) == 0:
        descrLines = ["-"]
    for descrLine in descrLines:
        print(formatstr %  (opts, argType, envVar, descrLine))
        argType = ""
        envVar = ""
        opts = ""
        
if len(sys.argv) < 2:
    print("usage: cmd [ go_file.go ... ]")
    exit(1)

cmd=sys.argv[1]

print("running command: ", cmd, file=sys.stderr)
output = os.popen(cmd).read()
print("output: ", output, file=sys.stderr)

state = "start"

arglist = []
for line in output.splitlines():
    print (state, line, file=sys.stderr)    
    if line.strip() == "":
        if state!="nop":
            print("")
            state = "nop"
        continue
    if line.startswith("Available Commands:"):
        state = "available"
        print("**"+line+"**\n")
        continue
    if line.startswith("Usage:"):
        state = "usage"
        print("**"+line+"**\n")
        continue
    if line.startswith("Flags:"):
        state = "flg"
        print("**"+line+"**\n")
        continue
    if state == "usage" or state=="available":
        print("* %s" % line.strip())
    if state == "start":
        print("%s\n\n" % line)
        state = "nop"
    if state == "flg":
        # The description column of the help output starts right after the
        # longest flag name, so the line is parsed by its shape instead of by
        # fixed offsets: a single long flag name would otherwise shift the
        # columns and cut every name and description of the command in two.
        flagMatch = FLAG_RE.match(line)
        if not flagMatch:
            print("UNPARSED FLAG LINE", line, file=sys.stderr)
            continue
        shortOpt = flagMatch.group(1) or ""
        longOpt = flagMatch.group(2)
        optType = ""
        descr = (flagMatch.group(3) or "").strip()
        if descr:
            parts = re.split("\\s{2,}", descr, maxsplit=1)
            if parts[0] in FLAG_TYPES:
                optType = parts[0]
                descr = parts[1].strip() if len(parts) > 1 else ""
        print("short=", shortOpt, "long=", longOpt, "type=", optType, "descr=", descr, file=sys.stderr)
        arglist.append((shortOpt, longOpt, optType, descr))

varmap = {}
typesmap = {}
for gofile in sys.argv[2:]:
    print ("processing file ", gofile, file=sys.stderr)
    with open(gofile) as fd:
        output = fd.read()
        for line in output.splitlines():
            line = line.strip()
            #if re.match("^[_a-z][a-zA-Z0-9_]*\\.Flags\\(\\)\\.[A-Z][A-Za-z0-9]*Var\\(.*$", line):
            if re.match("^[_a-z][a-zA-Z0-9_]*\\.Flags\\(\\)\\.[A-Z][A-Za-z0-9_]*Var.*$", line):
                print("MATCHED", line,  file=sys.stderr)
                argMatch = re.search("\"([a-z][a-z0-9-]+)\"", line)
                if not argMatch:
                    continue
                argName = argMatch.group(1)
                print("ARG", argName,  file=sys.stderr)
                argMatch = re.search("env.(Strings?|ParseNum|ParseBool|ParseDuration|ParseInt64|ParseFloat64|ParseFloat)FromEnv\\(\"([A-Z0-9_]+)\"", line)
                if not argMatch:
                    continue
                varName = argMatch.group(2)
                print("VAR", varName,  file=sys.stderr)
                varType = argMatch.group(1)
                if varType == "ParseBool":
                    typesmap[argName]="bool"
                varmap[argName] = varName
                
                
formatstr = f"| %-39s | %-11s | %-44s | %-60s |"

printline(formatstr, "", "Argument",  "Type", "Environment Variable", "Description")
printline(formatstr, "", "---------------------------------------", "-----------", "--------------------------------------------", "------------------------------------------------------------")

for arg in arglist:
    descr = arg[3].replace("|",",")
    envVar = ""
    argName = arg[1].removeprefix("--")
    if argName in varmap:
        envVar = varmap[argName]
    typeName = arg[2]
    if argName in typesmap:
        typeName = typesmap[argName]
    
    printline(formatstr, arg[0], arg[1], typeName, envVar, descr)
    # print("|", arg[0], arg[1], " | ", rg[2], " | ", envVar, " | ", descr, " |")


