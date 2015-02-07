# Powerstrip mode

Using powerstrip mode of socketplane is simple and easy. If you are not using vagrant run "socketplane install" from the base directory of the socketplane workspace or just run scripts/install.sh to setup everything. If you are using vagrant no other extra steps are required. 

Once socketplane is installed run new containers using the well known docker commands as below example shows:

     sudo DOCKER_HOST=localhost:2375 docker run -itd ubuntu

If you want the containers to connect to different network just add a special environment variable as follows:

     sudo DOCKER_HOST=localhost:2375 docker run -e SP_NETWORK=test -itd ubuntu

The above commands assumes that you have already created a network named "test" using the already existing socketplane commands.