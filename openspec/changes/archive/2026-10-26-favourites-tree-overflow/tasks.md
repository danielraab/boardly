# Tasks

## 1. Favourites page

- [x] 1.1 Replace `INDENT_REM` with a `--indent` CSS variable (0.75rem, 1.5rem from `sm`) used by the row padding and the drop indicator; verify the existing drag-and-drop tests pass and a test asserts the row padding uses the variable
- [x] 1.2 Wrap the tree list in an `overflow-x-auto` container with the list `w-max min-w-full`, make each row at least as wide as the frame (container query) and give the name box a minimum width (6rem, 8rem from `sm`); verify a test for the structure and that all favourites page tests pass

## 2. Explorer widget

- [x] 2.1 Wrap the explorer list in an `overflow-x-auto` container with the list `w-max min-w-full` and give names a minimum width; verify the explorer tests pass and a test asserts the scroll container

## 3. Maximum name width

- [x] 3.1 Give names on the favourites page and in the explorer a maximum width of 24rem with the full name as `title`; verify tests for the class and the tooltip

- [x] 3.2 Put a link's tags in a box that gives way before the name and shows only whole tags, with all tags as tooltip; verify a test for the box and the tooltip

## 4. Verification

- [x] 4.1 Run frontend lint, typecheck and `vitest` and verify they pass
- [x] 4.2 Run the app with a ten-level tree at 375px and 1200px: names visible, tree scrolls sideways, page doesn't, buttons not overlapping, drag and drop works while scrolled, and a narrow explorer widget scrolls, a 300-character title ends with "…" at 24rem, a link with twelve tags keeps its name and shows only whole tags; check light and dark themes
