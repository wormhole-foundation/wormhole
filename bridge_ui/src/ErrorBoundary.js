import { Typography } from "@material-ui/core";
import React from "react";

export default class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true };
  }

  componentDidCatch(error, errorInfo) {
    console.error(error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <Typography variant="h5" style={{ textAlign: "center", marginTop: 24 }}>
          An unexpected error has occurred. Please refresh the page.
        </Typography>
      );
    }

    return this.props.children;
  }
}
