function addTask() {
  var el = document.querySelector('input[name=desc]');
  var data = { desc: el.value };
  el.value = "";
  var xhr = {
    url: 'api/task',
    method: 'POST',
    credentials: 'include',
    body: JSON.stringify(data),
    cache: 'no-cache',
  };
  fetch(xhr.url, xhr).then(response => response.json()).then(data => {
    if (data.error) {
      setNotice('#FEE', 'Failed: ' + data.error);
    } else if (data.task) {
      var li = html('<li class="" onclick="toggle(this)">');
      li.id = 'task' + data.task.id;
      li.innerText = data.task.desc; // use innerText so it's escaped!
      document.querySelector('#tasks').appendChild(li);
      setNotice('#EFE', 'Added task ' + data.task.desc);
    } else {
      setNotice('#FEE', 'Server error');
      console.error('Bad response: ', data);
    }
  })

  return false;
}

function toggle(el) {
  var id = el.id.substr(4); // skip the "task" prefix
  var xhr = {
    url: 'api/task/' + id,
    method: 'POST',
    credentials: 'include',
    body: '{"toggle":true,"id":"' + id + '"}',
    cache: 'no-cache',
  };
  fetch(xhr.url, xhr).then(response => response.json()).then(data => {
    if (data.error) {
      setNotice('#FEE', 'Failed: ' + data.error);
    } else if (data.task) {
      el.classList.toggle('done', data.task.done);
      setNotice('#EFE', 'Updated!')
    } else {
      setNotice('#FEE', 'Server error');
      console.error('Bad response: ', data);
    }
  });
}

// Updates the notice div.
function setNotice(col, msg) {
  var el = document.querySelector('#notice');
  el.innerText = msg;
  el.style.background = col;
}

// Creates an html element from a string.
function html(content) {
  var div = document.createElement('div');
  div.innerHTML = content;
  return div.firstChild;
}
