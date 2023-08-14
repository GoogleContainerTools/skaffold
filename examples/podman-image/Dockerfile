FROM ubuntu
RUN apt-get update -y
RUN apt-get install curl podman newuid -y
RUN useradd -ms /bin/bash newuser
USER newuser
WORKDIR /home/newuser


