import { Container } from "@material-ui/core";
import HeaderText from "../HeaderText";
import TVLStats from "./TVLStats";
import VolumeStats from "./VolumeStats";

const StatsRoot = () => {
  return (
    <Container maxWidth="lg">
      <Container maxWidth="md">
        <HeaderText white>Stats</HeaderText>
      </Container>
      <TVLStats />
      <VolumeStats />
    </Container>
  );
};

export default StatsRoot;
