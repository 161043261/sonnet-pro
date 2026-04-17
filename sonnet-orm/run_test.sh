#!/bin/bash
set -eou pipefail

cur=$PWD
for item in "$cur"/day*/
do
    echo "$item"
    cd "$item"
    go test lark_orm/... 2>&1 | grep -v warning
done
