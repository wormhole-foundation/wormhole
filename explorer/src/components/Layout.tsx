import React from "react";
import Footer from "./Footer";
import NavBar from "./Navbar";

const Layout: React.FC = ({ children }) => (
  <main>
    <NavBar />
    {children}
    <Footer />
  </main>
);

export default Layout;
