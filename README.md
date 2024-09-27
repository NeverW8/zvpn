# zvpn
VPN Switcher done in a learning fashion

This was made to learn a bit and to solve a problem I had in a quick fashion.
It's not the best and there's better ways to do it, but I made it for me :)

---
Create a directory in your root folder called .zvpn
> sudo mkdir /root/.zvpn

Copy your ovpn files to the directory:
> sudo cp /your/ovpn/files /root/.zvpn

In your bashrc, zshrc or bash_profile add the following alias:
> alias zvpn="sudo zvpn"

Run like this:
> zvpn --start|stop|log|status
