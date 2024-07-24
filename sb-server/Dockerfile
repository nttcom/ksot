FROM python:3.11-slim-bookworm

WORKDIR /work/sb-server
COPY ./main.py ./
COPY ./requirements.txt ./
COPY ./connect.json ./
RUN pip install --upgrade pip
RUN pip install -r requirements.txt
CMD [ "main.py" ]
ENTRYPOINT ["python"]