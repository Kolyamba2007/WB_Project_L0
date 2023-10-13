import React, { useState } from 'react';
import './App.css';

function App() {
  const[data, setData] = useState('')

  const [inputData, setInputData] = useState('');

  const handleInputChange = (event) => {
    setInputData(event.target.value);
  }

  const handleSubmit = () => {
    fetch(`/api/?order_uid=${inputData}`).then(response => {
      const contentType = response.headers.get("content-type");
      if (contentType && contentType.indexOf("application/json") !== -1) {
        return response.json().then(data => setData(JSON.stringify(data, undefined, 2)));
      } else {
        return response.text().then(text => setData(text));
      }
    });
  }

  return (
    <div className="App">
      <header className="App-header">
        <h1>
          Данные по заказам
        </h1>
        <input
          type="text"
          value={inputData}
          onChange={handleInputChange}
          placeholder="Введите id заказа"
        />
        <button onClick={handleSubmit}>Получить данные</button>
        <div>
          <strong>Результат:</strong>
          <p>
          {
            data
          }
          </p>
        </div>
      </header>
    </div>
  );
}

export default App;