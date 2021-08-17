pipeline {
    agent none
    stages {
        stage('Parallel') {
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
                        sh "git show HEAD"
                    }
                    post {
                        success {
                            gerritCheck checks: ['jenkins:test': 'SUCCESSFUL']
                        }
                        unsuccessful {
                            gerritCheck checks: ['jenkins:test': 'FAILED']
                        }
                    }
                }
            }
        }
    }
}
