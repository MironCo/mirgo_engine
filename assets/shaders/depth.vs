#version 330

in vec3 vertexPosition;

uniform mat4 mvp;

out float fragDepth;

void main()
{
    gl_Position = mvp * vec4(vertexPosition, 1.0);
    fragDepth = gl_Position.z / gl_Position.w;
}
