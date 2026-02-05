#version 330

// Minimal fragment shader for shadow pass
// OpenGL writes depth automatically, we just need a valid color output
out vec4 finalColor;

void main()
{
    finalColor = vec4(1.0);
}