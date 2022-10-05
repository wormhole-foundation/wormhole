import axios from 'axios';

axios({
    method: 'post',
    url: 'http://127.0.0.1:5003/gas',
    timeout: 100000,    // 4 seconds timeout
    data: {}
  })
  .then(response => {console.log(response)})
  .catch(error => console.error('timeout exceeded'))