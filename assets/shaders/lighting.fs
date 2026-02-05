#version 330

in vec3 fragPosition;
in vec2 fragTexCoord;
in vec4 fragColor;
in vec3 fragNormal;

uniform sampler2D texture0;
uniform vec4 colDiffuse;
uniform vec3 viewPos;
uniform vec4 ambient;
uniform vec3 lightDir;
uniform vec4 lightColor;

out vec4 finalColor;

void main()
{
    vec3 normal = normalize(fragNormal);
    vec3 viewDir = normalize(viewPos - fragPosition);

    // Directional light
    vec3 lightDirection = normalize(lightDir);
    float diff = max(dot(normal, lightDirection), 0.0);

    // Specular (Blinn-Phong)
    vec3 halfwayDir = normalize(lightDirection + viewDir);
    float spec = pow(max(dot(normal, halfwayDir), 0.0), 32.0);

    vec3 diffuse = diff * lightColor.rgb;
    vec3 specular = spec * lightColor.rgb * 0.5;

    // Combine lighting
    vec3 lighting = ambient.rgb + diffuse + specular;

    vec3 result = lighting * colDiffuse.rgb * fragColor.rgb;

    finalColor = vec4(result, colDiffuse.a);
}
