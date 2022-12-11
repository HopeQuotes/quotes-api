git add .
echo -n "enter commit message : "
read -r message
git commit -m "$message"
git push
