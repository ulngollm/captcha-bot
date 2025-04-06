include .env

build:
	CGO_ENABLED=0 go build -o bot

start:
	nohup ./bot > log &

deploy:
	ssh ${SERVER} "mkdir -p ${DEPLOY_DIR}"
	scp bot ${SERVER}:${DEPLOY_DIR}
	scp .env.prod ${SERVER}:${DEPLOY_DIR}/.env