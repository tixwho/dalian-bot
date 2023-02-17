if test $( pgrep -f dalian | wc -l ) -eq 0
then
        echo "dalian process not exist"
else
        echo "dalian process exist, terminating..."
        pkill -2 "dalian"
        echo "dalian process succesfully terminated!"
fi
