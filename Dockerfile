FROM python:3.10

RUN mkdir /app
ADD . /app/
WORKDIR /app

RUN apt-get -y update
RUN apt-get update && apt-get install -y python3 python3-pip
RUN pip3 install pipenv
RUN pipenv install

ENTRYPOINT ["pipenv", "run", "python", "main.py"] 
