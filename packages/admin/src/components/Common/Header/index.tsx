import Link from "next/link";
import { HeaderContainer } from "./styled";

export default function Header(): JSX.Element {
  return (
    <HeaderContainer>
      <Link href={"/"}>Orakl Admin</Link>
    </HeaderContainer>
  );
}
