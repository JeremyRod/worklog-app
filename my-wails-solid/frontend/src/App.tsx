import type { Component } from 'solid-js';

import Form from './loginForm';

import logo from './logo.svg';
import styles from './App.module.css';

const App: Component = () => {
  return (
    <div>
      <div>      
        <Form />
      </div>  
    </div>

  );
};

export default App;
