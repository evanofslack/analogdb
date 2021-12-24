FROM python:3.8

RUN mkdir /app
ADD scraper /app/
WORKDIR /app

#Install the dependencies
RUN apt-get -y update
RUN apt-get update && apt-get install -y python3 python3-pip
RUN pip3 install pipenv
RUN pipenv install

#Run the command
CMD python main.py
