#!/bin/sh

current_time=$(date +%s)
stop_failing_time=$STOP_FAILING_TIME

echo $current_time
echo "========"
echo $stop_failing_time
echo "========"

if [[ $current_time -le $stop_failing_time ]]; then
   echo "current time less than stop failing time, container will exit with error"
   exit 1
fi
while :
do
	echo "Hello world!!!! - current time greater than stop failing time!"
	sleep 2
done