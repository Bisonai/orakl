import CachedIcon from "@mui/icons-material/Cached";
import { Icon } from "@mui/material";
const RefreshIcon = ({ onRefresh }: { onRefresh: () => void }) => {
  const handleRefresh = () => {
    onRefresh();
  };
  console.log("RefreshIcon");
  return (
    <div onClick={handleRefresh}>
      <Icon
        component={CachedIcon}
        style={{ fontSize: "36px", cursor: "pointer" }}
        onMouseEnter={(e: React.MouseEvent<SVGSVGElement, MouseEvent>) => {
          const target = e.currentTarget;
          target.style.color = "#858585";
        }}
        onMouseLeave={(e: React.MouseEvent<SVGSVGElement, MouseEvent>) => {
          const target = e.currentTarget;
          target.style.color = "white";
        }}
      />
    </div>
  );
};

export default RefreshIcon;
