current_path=$(realpath .)
echo "Current Path: ${current_path}"
cd "apps/dalian/dalian-deploy"
nohup apps/dalian/dalian-deploy/dalian-next &
echo 'dalian delopyment completed!'
