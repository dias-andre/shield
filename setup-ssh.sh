TOOL=podman
SSHUSER=sshuser
CONTAINER_NAME=shield
REPOSITORY=localhost # can be ghcr.io
KEYFILE=my-key.pem

ssh-keygen -t ed25519 -f ./$KEYFILE -N "" -C "$SSHUSER"

$TOOL run -d -p 2222:22 --name $CONTAINER_NAME $REPOSITORY/ssh-server:latest

sleep 3

if [ "$($TOOL inspect -f '{{.State.Running}}' $CONTAINER_NAME 2>/dev/null)" == "false" ]; then
  echo "The $CONTAINER_NAME is not running!"
  exit 1
fi

$TOOL exec -t $CONTAINER_NAME mkdir -p /home/$SSHUSER/.ssh
cat ./$KEYFILE.pub | $TOOL exec -i $CONTAINER_NAME sh -c "cat >> /home/$SSHUSER/.ssh/authorized_keys"

$TOOL exec $CONTAINER_NAME chown -R $SSHUSER /home/$SSHUSER/.ssh
$TOOL exec $CONTAINER_NAME chmod 700 /home/$SSHUSER/.ssh
$TOOL exec $CONTAINER_NAME chmod 600 /home/$SSHUSER/.ssh/authorized_keys

echo "To connect with server, use: 'ssh -i ./$KEYFILE -p 2222 $SSHUSER@localhost'"
