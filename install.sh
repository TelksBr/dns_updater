#!/bin/bash

mkdir /root/dns_service
wget -O /root/dns_service/dns_updater 
chmod +x /root/dns_service/dns_updater

SERVICE_NAME="api-v2-server"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
BINARY_PATH="/root/dns_service/dns_updater"
WORKING_DIR=/root"/dns_service"
USER="root"  # Substitua pelo nome do usuário que executará o serviço
GROUP="root" # Substitua pelo grupo do usuário

# Função para criar o arquivo de serviço
create_service_file() {
    echo "Criando o arquivo de serviço ${SERVICE_FILE}..."

    tee ${SERVICE_FILE} >/dev/null <<EOF
[Unit]
Description=API Server
After=network.target

[Service]
ExecStart=${BINARY_PATH}
WorkingDirectory=${WORKING_DIR}
Restart=always
User=${USER}
Group=${GROUP}


[Install]
WantedBy=multi-user.target
EOF
}

# Função para recarregar systemd e iniciar o serviço
reload_and_start_service() {
    echo "Recarregando systemd..."
    systemctl daemon-reload

    echo "Iniciando o serviço ${SERVICE_NAME}..."
    systemctl start ${SERVICE_NAME}

    echo "Habilitando o serviço ${SERVICE_NAME} para inicialização automática..."
    systemctl enable ${SERVICE_NAME}

    echo "Status do serviço ${SERVICE_NAME}:"
    systemctl status ${SERVICE_NAME}
}

# Executa as funções
create_service_file
reload_and_start_service

echo "Configuração do serviço ${SERVICE_NAME} concluída!"