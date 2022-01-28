FROM eth-node

ADD --chown=node:node truffle-config.js .
# ACA decimal is 12, add an extra 6 zeros to complement the precision
RUN sed -i 's/value: 530,/value: 530000000,/g' scripts/deploy_test_token.js