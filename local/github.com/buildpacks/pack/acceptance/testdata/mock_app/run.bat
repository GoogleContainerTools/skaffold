@echo off

set port=8080
if [%1] neq [] set port=%1

C:\util\server.exe -p %port% -g "%cd%\*-deps\*-dep, c:\contents*.txt"


