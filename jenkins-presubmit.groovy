final kubeCleanup = "kubectl delete --namespace=\$DEPLOY_NS service,statefulset,configmap,pod --all"

pipeline {
    agent none
    stages {
        stage('Parallel') {
            failFast true
            parallel {
                stage('Test') {
                    agent {
                        node {
                            label ""
                            customWorkspace '/home/ci/wormhole'
                        }
                    }
                    steps {
                        gerritCheck checks: ['jenkins:test': 'RUNNING'], message: "Running on ${env.NODE_NAME}"

                        echo "Gerrit change: ${GERRIT_CHANGE_URL}"
                        echo "Tilt progress dashboard: https://${DASHBOARD_URL}"

                        sh """
                        kubectl config set-context ci --namespace=$DEPLOY_NS
                        kubectl config use-context ci
                        """

                        sh kubeCleanup

                        timeout(time: 60, unit: 'MINUTES') {
                            sh "tilt ci -- --ci --namespace=$DEPLOY_NS --num=1"
                        }

                        timeout(time: 1, unit: 'MINUTES') {
                            sh "make node"
                        }
                    }
                    post {
                        success {
                            gerritCheck checks: ['jenkins:test': 'SUCCESSFUL']
                        }
                        unsuccessful {
                            gerritCheck checks: ['jenkins:test': 'FAILED']
                        }
                        cleanup {
                            sh kubeCleanup
                        }
                    }
                }
                stage('Lint') {
                    agent {
                        node {
                            label ""
                            customWorkspace '/home/ci/wormhole'
                        }
                    }
                    steps {
                        gerritCheck checks: ['jenkins:linters': 'RUNNING'], message: "Running on ${env.NODE_NAME}"

                        echo "Gerrit change: ${GERRIT_CHANGE_URL}"

                        timeout(time: 1, unit: 'MINUTES') {
                            sh "make generate"
                            sh "./lint.sh"
                        }
                    }
                    post {
                        success {
                            gerritCheck checks: ['jenkins:linters': 'SUCCESSFUL']
                        }
                        unsuccessful {
                            gerritCheck checks: ['jenkins:linters': 'FAILED']
                        }
                    }
                }
            }
            post {
                success {
                    gerritReview labels: [Verified: 1]
                }
                unsuccessful {
                    gerritReview labels: [Verified: -1]
                }
            }
        }
    }
}
