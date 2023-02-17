current_path=$(realpath .)
echo "Current Path: ${current_path}"
cd "apps/dalian/dalian-deploy"
screen -dmS dalian ./dalian-next
echo "Dalian is now ONLINE."
